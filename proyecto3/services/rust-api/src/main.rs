use axum::{
    body::Bytes,
    extract::{DefaultBodyLimit, State},
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{get, post},
    Json, Router,
};
use chrono::{DateTime, Datelike, SecondsFormat, Utc};
use regex::Regex;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::{
    env,
    sync::{Arc, LazyLock},
    time::Duration,
};
use tokio::sync::Semaphore;
use uuid::Uuid;

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Report {
    country: String,
    warplanes_in_air: u32,
    warships_in_water: u32,
    timestamp: String,
}
static TIMESTAMP: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:[0-5]\d(?:\.\d{1,6})?(?:Z|[+-]\d{2}:\d{2})$")
        .unwrap()
});
fn parse_report(body: &[u8]) -> Result<Report, String> {
    let mut report: Report = serde_json::from_slice(body).map_err(|_| "JSON o campos inválidos")?;
    if !["USA", "RUS", "CHN", "ESP", "GTM"].contains(&report.country.as_str()) {
        return Err("país inválido".into());
    }
    if report.warplanes_in_air > 50 || report.warships_in_water > 30 {
        return Err("cantidades fuera de rango".into());
    }
    if !TIMESTAMP.is_match(&report.timestamp) {
        return Err("timestamp inválido o precisión mayor a seis decimales".into());
    }
    let timestamp = DateTime::parse_from_rfc3339(&report.timestamp)
        .map_err(|_| "fecha inválida")?
        .with_timezone(&Utc);
    if !(1..=9999).contains(&timestamp.year()) {
        return Err("año fuera de rango".into());
    }
    report.timestamp = timestamp.to_rfc3339_opts(SecondsFormat::AutoSi, true);
    Ok(report)
}
#[derive(Clone)]
struct App {
    client: reqwest::Client,
    go_url: String,
    run: String,
    slots: Arc<Semaphore>,
}
fn error(code: StatusCode, message: &str, id: &str) -> Response {
    (
        code,
        Json(json!({"error": {"code": code.as_str(), "message": message}, "event_id": id})),
    )
        .into_response()
}
async fn ingest(
    State(app): State<App>,
    headers: HeaderMap,
    body: Result<Bytes, axum::extract::rejection::BytesRejection>,
) -> Response {
    let _permit = match app.slots.try_acquire() {
        Ok(p) => p,
        Err(_) => {
            return error(
                StatusCode::TOO_MANY_REQUESTS,
                "concurrencia máxima alcanzada",
                "",
            )
        }
    };
    let content_type = headers
        .get("content-type")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .split(';')
        .next()
        .unwrap_or("")
        .trim();
    if content_type != "application/json" {
        return error(
            StatusCode::UNSUPPORTED_MEDIA_TYPE,
            "se requiere application/json",
            "",
        );
    }
    let body = match body {
        Ok(value) => value,
        Err(rejection) => {
            return error(rejection.status(), "cuerpo inválido o demasiado grande", "")
        }
    };
    let report = match parse_report(&body) {
        Ok(report) => report,
        Err(message) => return error(StatusCode::BAD_REQUEST, &message, ""),
    };
    let id = match headers.get("idempotency-key") {
        Some(value) => match value
            .to_str()
            .ok()
            .filter(|s| s.len() == 36)
            .and_then(|s| Uuid::parse_str(s).ok())
        {
            Some(id) => id,
            None => return error(StatusCode::BAD_REQUEST, "Idempotency-Key debe ser UUID", ""),
        },
        None => Uuid::new_v4(),
    }
    .to_string();
    let run = match headers.get("x-run-id") {
        Some(value) => match value.to_str() {
            Ok(s) => s,
            Err(_) => return error(StatusCode::BAD_REQUEST, "X-Run-ID inválido", &id),
        },
        None => &app.run,
    };
    if run.is_empty()
        || run.len() > 64
        || !run
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b == b'_' || b == b'-')
    {
        return error(StatusCode::BAD_REQUEST, "X-Run-ID inválido", &id);
    }
    // The public client cannot extend the internal five-second deadline.
    let deadline = Utc::now().timestamp_millis() + 5000;
    let result = app
        .client
        .post(&app.go_url)
        .header("Idempotency-Key", &id)
        .header("X-Run-ID", run)
        .header("X-Request-Deadline", deadline.to_string())
        .json(&report)
        .send()
        .await;
    let response = match result {
        Ok(response) => response,
        Err(e) => {
            return error(
                if e.is_timeout() {
                    StatusCode::GATEWAY_TIMEOUT
                } else {
                    StatusCode::SERVICE_UNAVAILABLE
                },
                "servicio Go no disponible",
                &id,
            )
        }
    };
    let status = response.status();
    if status.as_u16() != 202 {
        let code = match status.as_u16() {
            400 | 413 | 415 | 429 | 500 | 503 | 504 => status.as_u16(),
            _ => 502,
        };
        return error(
            StatusCode::from_u16(code).unwrap(),
            "dependencia rechazó la solicitud",
            &id,
        );
    }
    let reply: serde_json::Value = match response.json().await {
        Ok(value) => value,
        Err(e) => {
            return error(
                if e.is_timeout() {
                    StatusCode::GATEWAY_TIMEOUT
                } else {
                    StatusCode::BAD_GATEWAY
                },
                "confirmación inválida",
                &id,
            )
        }
    };
    if reply["status"] != "accepted" || reply["event_id"] != id || reply["run_id"] != run {
        return error(StatusCode::BAD_GATEWAY, "confirmación inconsistente", &id);
    }
    (
        StatusCode::ACCEPTED,
        Json(json!({"status":"accepted","event_id":id,"run_id":run})),
    )
        .into_response()
}
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let app = App {
        client: reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()?,
        go_url: env::var("GO_API_URL").unwrap_or("http://go-api:8081/reports".into()),
        run: env::var("RUN_ID").unwrap_or("default".into()),
        slots: Arc::new(Semaphore::new(128)),
    };
    let router = Router::new()
        .route("/grpc-202100054", post(ingest))
        .route("/healthz", get(|| async { "ok" }))
        .layer(DefaultBodyLimit::max(4096))
        .with_state(app);
    let listener =
        tokio::net::TcpListener::bind(env::var("LISTEN_ADDR").unwrap_or("0.0.0.0:8080".into()))
            .await?;
    println!("mumn-api listening on {}", listener.local_addr()?);
    axum::serve(listener, router)
        .with_graceful_shutdown(async {
            #[cfg(unix)]
            {
                let mut term =
                    tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
                        .unwrap();
                tokio::select! { _ = tokio::signal::ctrl_c() => {}, _ = term.recv() => {} }
            }
            #[cfg(not(unix))]
            {
                let _ = tokio::signal::ctrl_c().await;
            }
        })
        .await?;
    Ok(())
}
#[cfg(test)]
mod tests {
    use super::*;
    fn valid() -> serde_json::Value {
        json!({"country":"CHN","warplanes_in_air":10,"warships_in_water":2,"timestamp":"2026-09-29T06:00:00-06:00"})
    }
    #[test]
    fn normalizes_timestamp() {
        let r = parse_report(&serde_json::to_vec(&valid()).unwrap()).unwrap();
        assert_eq!(r.timestamp, "2026-09-29T12:00:00Z");
    }
    #[test]
    fn rejects_invalid_fields() {
        for (field, value) in [
            ("country", json!("chn")),
            ("warplanes_in_air", json!(true)),
            ("warplanes_in_air", json!(10.0)),
            ("warplanes_in_air", json!(51)),
            ("warships_in_water", json!(-1)),
            ("timestamp", json!("2026-02-30T12:00:00Z")),
            ("timestamp", json!("2026-09-29T12:00:00.1234567Z")),
            ("extra", json!(1)),
        ] {
            let mut value_report = valid();
            value_report[field] = value;
            assert!(
                parse_report(&serde_json::to_vec(&value_report).unwrap()).is_err(),
                "{field}"
            );
        }
        let mut report = valid();
        report.as_object_mut().unwrap().remove("warships_in_water");
        assert!(parse_report(&serde_json::to_vec(&report).unwrap()).is_err());
    }
    #[test]
    fn accepts_bounds() {
        for (planes, ships) in [(0, 0), (50, 30)] {
            let mut r = valid();
            r["warplanes_in_air"] = json!(planes);
            r["warships_in_water"] = json!(ships);
            assert!(parse_report(&serde_json::to_vec(&r).unwrap()).is_ok());
        }
    }
}

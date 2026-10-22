FROM grafana/grafana:13.0.2
USER root
RUN mkdir -p /opt/grafana/plugins && grafana cli --pluginsDir /opt/grafana/plugins plugins install yesoreyeram-infinity-datasource 4.1.1 && chown -R 472:472 /opt/grafana/plugins
ENV GF_PATHS_PLUGINS=/opt/grafana/plugins
USER 472

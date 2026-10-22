#!/usr/bin/env python3
"""Build and deploy the Operator demo only to the dedicated kind cluster."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys

ROOT=Path(__file__).resolve().parents[1]
TOOLS=ROOT/'artifacts/tools'
KUBECONFIG=ROOT/'artifacts/kubeconfig'
CLUSTER='mumn-phase4'


def run(args,**kw):return subprocess.run([str(v) for v in args],check=True,**kw)

def capture(args):return subprocess.check_output([str(v) for v in args],text=True)


def main():
    run([sys.executable,ROOT/'scripts/k8s-tools.py'])
    env=os.environ.copy()
    if subprocess.run(['docker','buildx','version'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode:
        env['DOCKER_BUILDKIT']='0'
    for tag,path in [('mumn-go:phase4','services/go'),('mumn-operator:phase4','operator'),('mumn-grafana:phase4','monitoring')]:
        run(['docker','build','-t',tag,ROOT/path],env=env)
    clusters=capture([TOOLS/'kind','get','clusters']).splitlines()
    if CLUSTER not in clusters:
        run([TOOLS/'kind','create','cluster','--name',CLUSTER,'--image','kindest/node:v1.35.0','--kubeconfig',KUBECONFIG,'--wait','60s'])
    else:KUBECONFIG.write_text(capture([TOOLS/'kind','get','kubeconfig','--name',CLUSTER]))
    KUBECONFIG.chmod(0o600)
    k=[TOOLS/'kubectl','--kubeconfig',KUBECONFIG]
    assert capture(k+['config','current-context']).strip()=='kind-'+CLUSTER
    images=['mumn-go:phase4','mumn-operator:phase4','mumn-grafana:phase4','valkey/valkey:8.1.10-alpine','rabbitmq:4.1-alpine']
    for image in images:
        if subprocess.run(['docker','image','inspect',image],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode:run(['docker','pull',image])
        # Explicit platform avoids missing foreign-manifest blobs with Docker's containerd store.
        export=subprocess.Popen(['docker','image','save','--platform','linux/amd64',image],stdout=subprocess.PIPE)
        try:
            run(['docker','exec','-i',CLUSTER+'-control-plane','ctr','-n','k8s.io','images','import','--platform','linux/amd64','-'],stdin=export.stdout)
        finally:export.stdout.close()
        if export.wait()!=0:raise RuntimeError('Fallo exportando imagen '+image)
    run(k+['apply','-k',ROOT/'operator/config/default'])
    n=k+['-n','mumn']
    run(n+['rollout','restart','deployment/mumn-operator'])
    if not capture(n+['get','secret','grafana-admin','--ignore-not-found']).strip():
        secret={'apiVersion':'v1','kind':'Secret','metadata':{'name':'grafana-admin','namespace':'mumn'},'stringData':{'username':'admin','password':secrets.token_urlsafe(24)}}
        run(k+['apply','-f','-'],input=json.dumps(secret),text=True)
    config=capture(n+['create','configmap','fixture-data','--from-file='+str(ROOT/'tests/fixtures/reports.json'),'--dry-run=client','-o','json'])
    run(k+['apply','-f','-'],input=config,text=True)
    run(k+['apply','-f',ROOT/'operator/config/samples/monitoring_v1alpha1_valkeyinstance.yaml','-f',ROOT/'operator/config/samples/monitoring_v1alpha1_grafanainstance.yaml','-f',ROOT/'deploy/phase4/validation.yaml'])
    run(n+['rollout','status','deployment/mumn-operator','--timeout=120s'])
    # Local tags are reused after each build; restart pods to consume imported images.
    run(n+['rollout','restart','deployment/fixture-consumer'])
    if capture(n+['get','deployment','view-grafana','--ignore-not-found']).strip():
        run(n+['rollout','restart','deployment/view-grafana'])
    print('Aplicado al clúster local. Ejecutar make check-cluster para verificar.')


if __name__=='__main__':main()

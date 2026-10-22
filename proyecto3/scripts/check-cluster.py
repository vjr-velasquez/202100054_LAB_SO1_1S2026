#!/usr/bin/env python3
"""Acceptance tests restricted to the local kind-mumn-phase4 cluster."""
import base64
from datetime import datetime
from contextlib import contextmanager
import json
from pathlib import Path
import socket
import subprocess
import time
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

ROOT = Path(__file__).resolve().parents[1]
K = [str(ROOT/'artifacts/tools/kubectl'), '--kubeconfig', str(ROOT/'artifacts/kubeconfig')]
N = K + ['-n', 'mumn']
RESULTS = []


def command(*args):
    return subprocess.check_output(N+list(args), text=True, stderr=subprocess.PIPE).strip()


def obj(kind, name):
    return json.loads(command('get',kind,name,'-o','json'))


def wait(fn, timeout=90):
    end=time.monotonic()+timeout
    last=None
    while time.monotonic()<end:
        try:
            value=fn()
            if value:return value
        except (subprocess.CalledProcessError, URLError, ConnectionError, ValueError, KeyError) as e:last=e
        time.sleep(1)
    raise AssertionError(f'Condición no satisfecha: {last}')


def ready(kind,name):
    def current():
        o=obj(kind,name);s=o.get('status',{})
        return s.get('observedGeneration',0)>=o['metadata']['generation'] and s.get('readyReplicas',0)==1 and (kind!='deployment' or (s.get('updatedReplicas')==1 and s.get('replicas')==1 and s.get('availableReplicas')==1)) and (kind!='statefulset' or s.get('currentRevision')==s.get('updateRevision'))
    wait(current)


@contextmanager
def forward(remote):
    sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close()
    p=subprocess.Popen(N+['port-forward','service/view-grafana',f'{port}:{remote}'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    try:
        def connected():
            with socket.create_connection(('127.0.0.1',port),1):return True
        wait(connected,20)
        yield f'http://127.0.0.1:{port}'
    finally:p.terminate();p.wait(timeout=10)


def auth():
    data=obj('secret','grafana-admin')['data']
    return 'Basic '+base64.b64encode(base64.b64decode(data['username'])+b':'+base64.b64decode(data['password'])).decode()


def request(url,body=None,authorization=None):
    headers={'Content-Type':'application/json'}
    if authorization:headers['Authorization']=authorization
    return json.load(urlopen(Request(url,headers=headers,data=json.dumps(body).encode() if body is not None else None),timeout=10))


def count():return command('exec','store-valkey-0','--','valkey-cli','HGET','mumn:run:fixture','count')


def passed(message):RESULTS.append(message);print('OK:',message,flush=True)


def queries(expected_title='M.U.M.N — CHN'):
    with forward(3000) as url:
        a=auth()
        def provisioned():
            d=request(url+'/api/dashboards/uid/mumn-202100054',authorization=a)['dashboard']
            return d if d['title']==expected_title else None
        d=wait(provisioned,120)
        assert d['title']==expected_title,d['title']
        assert len(d['panels'])==11
        for i in (4,5,6):
            assert d['panels'][i]['options']['reduceOptions']['fields']=='/.*/', 'Stat debe incluir campos textuales'
        expected=[[30],[0],[8],[0],['20'],['5'],['China (CHN)'],[3],['CHN','USA','ESP'],['CHN','USA','ESP']]
        for i,panel in enumerate(d['panels']):
            t=panel['targets'][0]
            assert t['url_options']['method']=='GET'
            t['url']=t['url'].replace('${run:percentencode}','fixture').replace('${__from}','0').replace('${__to}','4102444800000')
            result=request(url+'/api/ds/query',{'queries':[t],'from':'0','to':'4102444800000'},a)['results']['A']
            assert result.get('status',200)==200 and not result.get('error'),result
            frames=result.get('frames',[]);assert frames,panel['title']
            values=frames[0]['data']['values']
            if i<10 and i not in (4,5):assert values[0]==expected[i],(panel['title'],values)
            if i in (4,5):assert [2] in values and expected[i] in values,values
            if i==8:assert values[1]==[30,20,0]
            if i==9:assert values[1]==[8,5,0]
            if i==10:
                # Infinity may return either a wide frame or separate frames per series.
                all_values=[column for frame in frames for column in frame['data']['values']]
                assert [10,20,30] in all_values and [2,5,8] in all_values,all_values


def main():
    context=subprocess.check_output(K+['config','current-context'],text=True).strip()
    assert context=='kind-mumn-phase4','Este test solo puede usar kind-mumn-phase4'
    ready('deployment','mumn-operator');ready('statefulset','store-valkey');ready('deployment','view-grafana')
    wait(lambda:count()=='5')
    queries();passed('Los 11 paneles consultan el plugin real y devuelven todos los indicadores esperados')
    with forward(8084) as url:
        empty=request(url+'/stats?run=empty')
        assert empty['summary'][0]['max_planes'] is None and empty['summary'][0]['count']==0
        start=int(datetime.fromisoformat('2026-09-29T12:03:00+00:00').timestamp()*1000)
        filtered=request(url+f'/stats?run=fixture&from={start}')
        assert filtered['summary'][0]['count']==3 and len(filtered['series'])==1 and filtered['series'][0]['planes']==30
        try:request(url+'/stats?run=invalid/run');raise AssertionError('corrida inválida aceptada')
        except HTTPError as e:assert e.code==400
    passed('Corridas vacías e inválidas se distinguen de reportes cero')
    pvc=obj('pvc','data-store-valkey-0')['metadata']['uid']
    command('patch','valkeyinstance','store','--type=merge','-p',json.dumps({'spec':{'maxMemory':'192mb'}}))
    wait(lambda:'192mb' in obj('statefulset','store-valkey')['spec']['template']['spec']['containers'][0]['args'])
    ready('statefulset','store-valkey');assert count()=='5'
    passed('Actualizar el CR modifica Valkey y conserva reportes')
    old_pod=obj('pod','store-valkey-0')['metadata']['uid']
    command('delete','pod','store-valkey-0','--wait=false')
    def replaced():
        pod=obj('pod','store-valkey-0')
        return pod['metadata']['uid']!=old_pod and not pod['metadata'].get('deletionTimestamp') and pod['status'].get('containerStatuses',[{}])[0].get('ready',False)
    wait(replaced)
    wait(lambda:count()=='5');passed('Persistencia verificada al recrear el pod Valkey')
    old=obj('service','store-valkey')['metadata']['uid']
    command('delete','service','store-valkey')
    wait(lambda:obj('service','store-valkey')['metadata']['uid']!=old)
    passed('Operator reconstruye Service eliminado')
    generation=obj('deployment','view-grafana')['metadata']['generation']
    command('patch','grafanainstance','view','--type=merge','-p',json.dumps({'spec':{'dashboardTitle':'MUMN actualizado'}}))
    wait(lambda:obj('deployment','view-grafana')['metadata']['generation']>generation)
    wait(lambda:'MUMN actualizado' in obj('configmap','view-dashboard')['data']['mumn.json'])
    ready('deployment','view-grafana');queries('MUMN actualizado')
    passed('Actualizar GrafanaInstance reprovisiona el dashboard')
    command('delete','grafanainstance','view','--wait=true','--timeout=60s')
    wait(lambda:command('get','deployment','view-grafana','--ignore-not-found')=='')
    command('delete','valkeyinstance','store','--wait=true','--timeout=60s')
    wait(lambda:command('get','statefulset','store-valkey','--ignore-not-found')=='')
    assert obj('pvc','data-store-valkey-0')['metadata']['uid']==pvc
    command('apply','-f',str(ROOT/'operator/config/samples/monitoring_v1alpha1_valkeyinstance.yaml'),'-f',str(ROOT/'operator/config/samples/monitoring_v1alpha1_grafanainstance.yaml'))
    ready('statefulset','store-valkey');ready('deployment','view-grafana')
    wait(lambda:count()=='5');queries()
    assert obj('pvc','data-store-valkey-0')['metadata']['uid']==pvc
    passed('Eliminar/recrear ambos CR conserva el mismo PVC y restaura la visualización')
    (ROOT/'artifacts/phase4-results.json').write_text(json.dumps({'cluster':context,'passed':RESULTS},ensure_ascii=False,indent=2)+'\n')
    print(f'{len(RESULTS)} verificaciones aprobadas; GKE pendiente.')


if __name__=='__main__':main()

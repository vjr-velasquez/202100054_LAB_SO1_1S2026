#!/usr/bin/env python3
"""Pinned Linux/amd64 CLIs, installed only in ignored project artifacts."""
import hashlib
import io
from pathlib import Path
import tarfile
from urllib.request import urlopen

ROOT=Path(__file__).resolve().parents[1]
DEST=ROOT/'artifacts/tools'
DEST.mkdir(parents=True,exist_ok=True)


def download(url):
    with urlopen(url,timeout=60) as response:return response.read()


def verified(url,checksum_url,filename=None):
    manifest=download(checksum_url).decode()
    lines=manifest.splitlines()
    if filename:lines=[line for line in lines if line.split()[-1].lstrip('*')==filename]
    if len(lines)!=1:raise ValueError('No se encontró checksum único para '+url)
    digest=lines[0].split()[0]
    data=download(url)
    if hashlib.sha256(data).hexdigest()!=digest:raise ValueError('Checksum inválido: '+url)
    return data


def main():
    specs={
      'kind':('https://github.com/kubernetes-sigs/kind/releases/download/v0.33.0/kind-linux-amd64','https://github.com/kubernetes-sigs/kind/releases/download/v0.33.0/kind-linux-amd64.sha256sum',None),
      'kubectl':('https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl','https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl.sha256',None),
      'operator-sdk':('https://github.com/operator-framework/operator-sdk/releases/download/v1.42.3/operator-sdk_linux_amd64','https://github.com/operator-framework/operator-sdk/releases/download/v1.42.3/checksums.txt','operator-sdk_linux_amd64')}
    for name,args in specs.items():
        path=DEST/name
        # Existing files are rechecked against the published hash, without redownloading.
        manifest=download(args[1]).decode().splitlines()
        if args[2]:manifest=[line for line in manifest if line.split()[-1].lstrip('*')==args[2]]
        assert len(manifest)==1
        digest=manifest[0].split()[0]
        if not path.exists() or hashlib.sha256(path.read_bytes()).hexdigest()!=digest:
            path.write_bytes(verified(*args))
        path.chmod(0o755)
        print(name,'verificado')
    archive=DEST/'helm.tar.gz'
    url='https://get.helm.sh/helm-v3.19.0-linux-amd64.tar.gz'
    digest=download(url+'.sha256sum').decode().split()[0]
    if not archive.exists() or hashlib.sha256(archive.read_bytes()).hexdigest()!=digest:
        archive.write_bytes(verified(url,url+'.sha256sum'))
    with tarfile.open(fileobj=io.BytesIO(archive.read_bytes())) as bundle:
        data=bundle.extractfile('linux-amd64/helm').read()
    (DEST/'helm').write_bytes(data);(DEST/'helm').chmod(0o755)
    print('helm verificado')


if __name__=='__main__':main()

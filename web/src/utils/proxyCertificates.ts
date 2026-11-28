import type {CertificateRecord} from '../api/types'

// Advisory only: Nginx validates the actual SAN, validity and key pair at publish time.
export function previewProxyCertificates(inputs:string[], certificates:CertificateRecord[], now=Date.now()){
  const normalize=(value:string)=>value.trim().toLowerCase().replace(/:\d+$/,'').replace(/\.$/,'')
  const hosts=[...new Set(inputs.map(normalize).filter(Boolean))]
  return hosts.map(host=>{
    const candidates=certificates.filter(cert=>{
      const expires=Date.parse(cert.expires_at??'')
      if(!Number.isFinite(expires)||expires<=now||!cert.cert_path||!cert.key_path)return false
      const names=cert.domains?.length?cert.domains:[cert.domain]
      return names.some(value=>{
        const name=normalize(value)
        if(name===host)return true
        if(host.startsWith('*.')||!name.startsWith('*.'))return false
        const suffix=name.slice(1)
        const label=host.slice(0,-suffix.length)
        return host.endsWith(suffix)&&label.length>0&&!label.includes('.')
      })
    }).sort((a,b)=>Date.parse(b.expires_at??'')-Date.parse(a.expires_at??''))
    return {host,cert:candidates[0]}
  })
}

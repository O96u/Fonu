/** 各 ACME 服务商 EAB / 凭据申请入口（设置页展示） */
export type AcmeApplyLink = {
  label: string
  url: string
  linkText: string
}

export const ACME_APPLY_LINKS = {
  zerossl: {
    label: 'ZeroSSL',
    url: 'https://app.zerossl.com/developer',
    linkText: '打开 ZeroSSL Developer 申请 EAB',
  },
  google: {
    label: 'Google Trust Services',
    url: 'https://cloud.google.com/certificate-manager/docs/public-ca-tutorial',
    linkText: '查看 Google 公网 CA 与 EAB 申请说明',
  },
  sslcom: {
    label: 'SSL.com',
    url: 'https://www.ssl.com/how-to/generate-acme-credentials-for-reseller-and-enterprise-customers/',
    linkText: '查看 SSL.com ACME 凭据申请说明',
  },
  freessl: {
    label: 'FreeSSL',
    url: 'https://freessl.cn/automation/eab-manager',
    linkText: '打开 FreeSSL EAB 管理页',
  },
  actalis: {
    label: 'Actalis',
    url: 'https://www.actalis.com/Products/ACME-Credentials',
    linkText: '打开 Actalis ACME 凭据说明',
  },
} satisfies Record<string, AcmeApplyLink>

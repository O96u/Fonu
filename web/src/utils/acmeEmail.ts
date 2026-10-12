const ACME_EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function isValidAcmeEmail(raw: string): boolean {
  const email = raw.trim()
  if (!email || email.includes(' ')) return false
  return ACME_EMAIL_PATTERN.test(email)
}

export function validateAcmeEmailMessage(raw: string): string {
  const email = raw.trim()
  if (!email) {
    return '请先填写 ACME 邮箱，用于注册 ACME 账户（首次申请必填）'
  }
  if (!isValidAcmeEmail(email)) {
    return 'ACME 邮箱格式不正确，请填写类似 admin@example.com 的地址'
  }
  return ''
}

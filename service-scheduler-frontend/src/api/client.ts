export class APIClient {
  baseUrl: string

  constructor(baseUrl?: string) {
    this.baseUrl = (baseUrl || (import.meta.env.VITE_API_URL as string) || '/').replace(/\/+$/, '')
  }

  private async request(path: string, opts: RequestInit = {}) {
    const url = this.baseUrl + path
    const res = await fetch(url, opts)
    let body: any = null
    try {
      body = await res.json()
    } catch (_) {
      // ignore non-json
    }
    if (!res.ok) {
      const msg = (body && body.error) || body || res.statusText
      throw new Error(msg)
    }
    return body
  }

  get<T = any>(path: string): Promise<T> {
    return this.request(path, { method: 'GET' }) as Promise<T>
  }

  post<T = any>(path: string, data?: any): Promise<T> {
    return this.request(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }) as Promise<T>
  }
}

export const apiClient = new APIClient()

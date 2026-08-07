// Lightweight WebSocket helper for dealership real-time appointment events.
// Returns a cleanup function that closes the connection.

export function connectDealershipSocket(
  dealershipId: string,
  onMessage: (data: any) => void,
): () => void {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const host = window.location.hostname
  const url = `${protocol}://${host}:8080/ws?dealership_id=${encodeURIComponent(dealershipId)}`

  const ws = new WebSocket(url)
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data)
      if (data && data.type) onMessage(data)
    } catch {
      // ignore malformed messages
    }
  }
  ws.onerror = () => console.error('ws connect failed', url)

  return () => {
    ws.close()
  }
}

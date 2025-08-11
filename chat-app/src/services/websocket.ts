interface WebSocketMessage {
  id?: number
  username: string
  content: string
  server?: string
  timestamp?: string
}

export class ChatWebSocket {
  private ws: WebSocket | null = null
  private serverUrl: string
  private username: string
  private onMessage: (message: WebSocketMessage) => void
  private onConnect: () => void
  private onDisconnect: () => void
  private onError: (error: string) => void

  constructor(
    serverUrl: string,
    username: string,
    onMessage: (message: WebSocketMessage) => void,
    onConnect: () => void,
    onDisconnect: () => void,
    onError: (error: string) => void
  ) {
    this.serverUrl = serverUrl
    this.username = username
    this.onMessage = onMessage
    this.onConnect = onConnect
    this.onDisconnect = onDisconnect
    this.onError = onError
  }

  connect() {
    try {
      const wsUrl = `${this.serverUrl}/ws?username=${encodeURIComponent(this.username)}`
      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.onConnect()
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          
          if (typeof message.content === 'string' && message.content.startsWith('{')) {
            try {
              const parsedContent = JSON.parse(message.content)
              if (parsedContent.content) {
                message.content = parsedContent.content
                message.username = parsedContent.username || message.username
              }
            } catch (e) {
              console.log('Failed to parse nested JSON content:', e)
            }
          }
          
          this.onMessage(message)
        } catch (error) {
          console.error('Error parsing WebSocket message:', error)
        }
      }

      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.onDisconnect()
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        this.onError('Connection error occurred')
      }
    } catch (error) {
      console.error('Error creating WebSocket connection:', error)
      this.onError('Failed to create connection')
    }
  }

  sendMessage(content: string) {
    
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      const message = {
        username: this.username,
        content: content
      }
      this.ws.send(JSON.stringify(message))
    } else {
      this.onError('Connection is not open')
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}
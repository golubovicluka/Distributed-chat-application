import { useState, useEffect } from 'react'
import './App.css'
import { Login } from './components/Login'
import { ChatRoom } from './components/ChatRoom'
import { getOptimalServer, getChatHistory } from './services/api'
import { ChatWebSocket } from './services/websocket'

interface Message {
  id?: number
  username: string
  content: string
  server?: string
  timestamp?: string
}

function App() {
  const [isConnected, setIsConnected] = useState(false)
  const [username, setUsername] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [websocket, setWebsocket] = useState<ChatWebSocket | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<string>('')

  const handleConnect = async (inputUsername: string) => {
    try {
      setConnectionStatus('Getting optimal server...')
      const serverAddress = await getOptimalServer()

      setConnectionStatus('Loading chat history...')
      const history = await getChatHistory(serverAddress)
      setMessages(history || [])

      setConnectionStatus('Connecting to chat...')
      const ws = new ChatWebSocket(
        serverAddress,
        inputUsername,
        (message) => {
          const messageWithId = {
            ...message,
            id: message.id || Date.now() + Math.random(),
            timestamp: message.timestamp || new Date().toISOString()
          }
          setMessages(prev => [...prev, messageWithId])
        },
        () => {
          setConnectionStatus('')
          setIsConnected(true)
          setUsername(inputUsername)
        },
        () => {
          setIsConnected(false)
          setConnectionStatus('Disconnected')
        },
        (error) => {
          console.error('WebSocket error:', error)
          setConnectionStatus(`Error: ${error}`)
        }
      )

      ws.connect()
      setWebsocket(ws)
    } catch (error) {
      console.error('Connection error:', error)
      setConnectionStatus('Failed to connect. Please try again.')
    }
  }

  const handleSendMessage = (content: string) => {
    websocket?.sendMessage(content)
  }

  const handleDisconnect = () => {
    websocket?.disconnect()
    setWebsocket(null)
    setIsConnected(false)
    setUsername('')
    setMessages([])
    setConnectionStatus('')
  }

  useEffect(() => {
    return () => {
      websocket?.disconnect()
    }
  }, [websocket])

  if (!isConnected) {
    return (
      <div>
        <Login onConnect={handleConnect} />
        {connectionStatus && (
          <div className="fixed bottom-4 right-4 bg-background border rounded-lg p-3 shadow-lg">
            <p className="text-sm">{connectionStatus}</p>
          </div>
        )}
      </div>
    )
  }

  return (
    <ChatRoom
      username={username}
      messages={messages}
      onSendMessage={handleSendMessage}
      onDisconnect={handleDisconnect}
    />
  )
}

export default App

import { useState, useRef, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

interface Message {
  id?: number
  username: string
  content: string
  server?: string
  timestamp?: string
}

interface ChatRoomProps {
  username: string
  messages: Message[]
  onSendMessage: (message: string) => void
  onDisconnect: () => void
}

export function ChatRoom({ username, messages, onSendMessage, onDisconnect }: ChatRoomProps) {
  const [newMessage, setNewMessage] = useState("")
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newMessage.trim()) return

    onSendMessage(newMessage.trim())
    setNewMessage("")
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const formatTimestamp = (timestamp?: string) => {
    if (!timestamp) return ""
    const date = new Date(timestamp)
    return date.toLocaleTimeString()
  }

  return (
    <div className="min-h-screen bg-background p-4">
      <Card className="max-w-4xl mx-auto h-[calc(100vh-2rem)]">
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Chat Room</CardTitle>
            <p className="text-sm text-muted-foreground">Welcome, {username}!</p>
          </div>
          <Button variant="outline" onClick={onDisconnect}>
            Disconnect
          </Button>
        </CardHeader>
        <CardContent className="flex flex-col h-[calc(100vh-8rem)]">
          <div className="flex-1 overflow-y-auto space-y-3 p-4 bg-muted/50 rounded-lg mb-4">
            {messages && messages.length > 0 ? (
              messages.map((message, index) => {
              // Create a unique key using multiple fallbacks
              const messageKey = message.id ||
                `${message.username}-${message.timestamp}-${index}` ||
                `msg-${Date.now()}-${index}`

              return (
                <div
                  key={messageKey}
                  className={`flex flex-col space-y-1 ${message.username === username ? "items-end" : "items-start"
                    }`}
                >
                  <div
                    className={`max-w-[70%] rounded-lg px-3 py-2 ${message.username === username
                        ? "bg-primary text-primary-foreground"
                        : "bg-background border"
                      }`}
                  >
                    <p className="text-sm font-medium">{message.username}</p>
                    <p>{typeof message.content === 'string' ? message.content : JSON.stringify(message.content)}</p>
                    {message.timestamp && (
                      <p className="text-xs opacity-70 mt-1">
                        {formatTimestamp(message.timestamp)}
                      </p>
                    )}
                  </div>
                </div>
              )
            })
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground">
                <p>No messages yet. Start the conversation!</p>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          <form onSubmit={handleSubmit} className="flex space-x-2">
            <Input
              type="text"
              placeholder="Type your message..."
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              onKeyDown={handleKeyPress}
              className="flex-1"
            />
            <Button type="submit" disabled={!newMessage.trim()}>
              Send
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
interface ServerInfo {
  Address: string
  Load: number
}

interface HistoryMessage {
  id: number
  username: string
  content: string
  server: string
  timestamp: string
}

export async function getOptimalServer(): Promise<string> {
  try {
    const response = await fetch('http://localhost:9000/get')
    if (!response.ok) {
      throw new Error('Failed to get server information')
    }
    
    const data: ServerInfo = await response.json()
    return data.Address
  } catch (error) {
    console.error('Error getting optimal server:', error)
    throw error
  }
}

export async function getChatHistory(serverUrl: string): Promise<HistoryMessage[]> {
  try {
    const portMatch = serverUrl.match(/:(\d{4})\/?/)
    if (!portMatch) {
      throw new Error('Invalid server URL format')
    }
    
    const port = portMatch[1]
    const historyUrl = `http://localhost:${port}/history`
    const response = await fetch(historyUrl)
    
    if (!response.ok) {
      throw new Error('Failed to get chat history')
    }
    
    const data: HistoryMessage[] = await response.json()
    return data || []
  } catch (error) {
    console.error('Error getting chat history:', error)
    throw error
  }
}
import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import type { Transaction } from '@/components/dashboard/TransactionTable'

export function useWebSocket(url: string, onMessage: (tx: Transaction) => void) {
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null)
  const onMessageRef = useRef(onMessage)

  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    const connect = () => {
      console.log('Attempting WebSocket connection...')
      const socket = new WebSocket(url)

      socket.onopen = () => {
        console.log('WebSocket connected ✅')
        if (reconnectTimeout.current) {
          clearTimeout(reconnectTimeout.current)
        }
      }

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.id) {
            const tx: Transaction = data
            toast.success(`Transaction ${tx.status}`, {
              description: `Tx: ${tx.id.split('-')[0]} | Amount: $${tx.amount.toFixed(2)}`,
            })
            onMessageRef.current(tx)
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message', error)
        }
      }

      socket.onclose = () => {
        console.log('WebSocket disconnected ❌. Retrying in 3 seconds...')
        reconnectTimeout.current = setTimeout(() => {
          connect()
        }, 3000)
      }

      socket.onerror = (err) => {
        console.error('WebSocket error:', err)
        socket.close()
      }

      ws.current = socket
    }

    connect()

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current)
      }
      ws.current?.close()
    }
  }, [url])

  return ws
}

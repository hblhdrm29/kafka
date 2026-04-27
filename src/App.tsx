import { useState, useEffect } from 'react'
import { Toaster } from "@/components/ui/sonner"
import { Navbar } from "@/components/layout/Navbar"
import { StatCards } from "@/components/dashboard/StatCards"
import { TransactionTable, type Transaction } from "@/components/dashboard/TransactionTable"
import { useWebSocket } from "@/hooks/useWebSocket"
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'

function App() {
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [loading, setLoading] = useState(true)

  // Calculate stats
  const totalAmount = transactions.reduce((sum, tx) => sum + tx.amount, 0)
  const successCount = transactions.filter(tx => tx.status === 'PROCESSED' || tx.status === 'SUCCESS').length

  // Fetch initial transactions
  useEffect(() => {
    fetch('http://localhost/api/transactions')
      .then(res => res.json())
      .then(data => {
        if (data) {
          // ensure it's an array
          setTransactions(Array.isArray(data) ? data : [])
        }
        setLoading(false)
      })
      .catch(err => {
        console.error("Failed to fetch transactions:", err)
        setLoading(false)
      })
  }, [])

  // Listen for real-time Kafka messages
  useWebSocket('ws://localhost/ws', (newTx: Transaction) => {
    setTransactions(prev => {
      // Check if tx already exists (outbox might be processed fast)
      const exists = prev.find(t => t.id === newTx.id)
      if (exists) {
        return prev.map(t => t.id === newTx.id ? newTx : t)
      }
      return [newTx, ...prev]
    })
  })

  // Mock function to trigger a new transaction via API
  const handleCreateTransaction = async () => {
    const amount = parseFloat((Math.random() * 500 + 10).toFixed(2))
    try {
      const res = await fetch('http://localhost/api/transactions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          amount: amount,
          description: `Payment for Order #${Math.floor(Math.random() * 10000)}`
        })
      })
      const newTx = await res.json()
      // Add to UI immediately as pending
      setTransactions(prev => [newTx, ...prev])
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div className="min-h-screen bg-muted/40">
      <Navbar />
      <main className="flex-1 space-y-4 p-8 pt-6 max-w-6xl mx-auto">
        <div className="flex items-center justify-between space-y-2">
          <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
          <div className="flex items-center space-x-2">
            <Button onClick={handleCreateTransaction}>
              <Plus className="mr-2 h-4 w-4" /> New Transaction
            </Button>
          </div>
        </div>

        <StatCards 
          totalTransactions={transactions.length} 
          totalAmount={totalAmount} 
          successCount={successCount} 
        />
        
        <div className="grid gap-4 md:grid-cols-1">
          <TransactionTable transactions={transactions} loading={loading} />
        </div>
      </main>
      <Toaster position="top-right" />
    </div>
  )
}

export default App

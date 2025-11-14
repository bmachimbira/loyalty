export default function Dashboard() {
  return (
    <div className="container mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-card p-6 rounded-lg border">
          <h3 className="text-sm font-medium text-muted-foreground">Active Customers</h3>
          <p className="text-2xl font-bold mt-2">0</p>
        </div>
        <div className="bg-card p-6 rounded-lg border">
          <h3 className="text-sm font-medium text-muted-foreground">Events Today</h3>
          <p className="text-2xl font-bold mt-2">0</p>
        </div>
        <div className="bg-card p-6 rounded-lg border">
          <h3 className="text-sm font-medium text-muted-foreground">Rewards Issued</h3>
          <p className="text-2xl font-bold mt-2">0</p>
        </div>
        <div className="bg-card p-6 rounded-lg border">
          <h3 className="text-sm font-medium text-muted-foreground">Redemption Rate</h3>
          <p className="text-2xl font-bold mt-2">0%</p>
        </div>
      </div>
    </div>
  )
}

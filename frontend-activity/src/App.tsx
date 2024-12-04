import './App.css'
import Counter from './components/counter/Counter'
import Header from './components/header/Header'

function App() {
  const handleCountChange = (newCount: number) => {
    console.log("Counter value:", newCount);
  }
  return (
    <>
      <div>
      <Header />
      <Counter />
      <Counter 
        initialCount={10} 
        step={5} 
        onCountChange={handleCountChange} 
        cardStyle={{ backgroundColor: '#f8f9fa' }} 
        buttonLabels={{ increment: "Increase", decrement: "Decrease" }} 
      />
      </div>
    </>
  )
}

export default App

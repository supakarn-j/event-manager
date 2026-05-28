
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Home from './pages/home'
import StreamDetail from './pages/stream-detail'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/home" replace />} />
        <Route path="/home" element={<Home />} />
        <Route path="/streams/:streamName" element={<StreamDetail />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App

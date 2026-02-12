import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Home } from '@/pages/Home'
import { Preferences } from '@/pages/Preferences'
import { ReadingList } from '@/pages/ReadingList'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Home />} />
          <Route path="/preferences" element={<Preferences />} />
          <Route path="/reading-list" element={<ReadingList />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

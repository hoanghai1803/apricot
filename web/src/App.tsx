import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Home } from '@/pages/Home'
import { Preferences } from '@/pages/Preferences'
import { ReadingList } from '@/pages/ReadingList'

const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      { path: '/', element: <Home /> },
      { path: '/preferences', element: <Preferences /> },
      { path: '/reading-list', element: <ReadingList /> },
    ],
  },
])

export default function App() {
  return <RouterProvider router={router} />
}

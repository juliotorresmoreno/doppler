import React from 'react'
import { Route, Routes } from 'react-router-dom'
import withSession from '../hoc/withSession'
import { useAppSelector } from '../store/hooks'
import NotFoundPage from '../pages/NotFound'

const _App: React.FC = () => {
  const session = useAppSelector((state: any) => state.auth.session)

  return (
    <Routes>
      {session ? (
        <>
          <Route path="/" element={'Home'} />
          <Route path="*" element={<NotFoundPage />} />
        </>
      ) : (
        <>
          <Route path="*" element={<NotFoundPage />} />
        </>
      )}
    </Routes>
  )
}

const App = withSession(_App)

export default App

import React from 'react'
import { Route, Routes } from 'react-router-dom'
import withSession from '../hoc/withSession'
import { useAppSelector } from '../store/hooks'
import NotFoundPage from '../pages/NotFound'
import PreHomePage from '../pages/PreHomePage'
import HomePage from '../pages/HomePage'
import ServersPage from '../pages/ServersPage'
import HistoryPage from '../pages/HistoryPage'
import OrganizationsPage from '../pages/OrganizationsPage'
import PolicePage from '../pages/PolicePage'

const _App: React.FC = () => {
  const session = useAppSelector((state: any) => state.auth.session)

  return (
    <Routes>
      {session ? (
        <>
          <Route path="/" element={<HomePage />} />
          <Route path="/servers/" element={<ServersPage />} />
          <Route path="/history" element={<HistoryPage />} />
          <Route path="/organization" element={<OrganizationsPage />} />
          <Route path="/police" element={<PolicePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </>
      ) : (
        <>
          <Route path="/" element={<PreHomePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </>
      )}
    </Routes>
  )
}

const App = withSession(_App)

export default App

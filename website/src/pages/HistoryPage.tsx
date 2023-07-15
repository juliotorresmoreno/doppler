import React from 'react'
import PageTemplate from '../layouts/PageTemplate'

const HistoryPage: React.FC = () => {
  const header = {
    title: 'Servers',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>HistoryPage</PageTemplate>
    </>
  )
}

export default HistoryPage

import React from 'react'
import PageTemplate from '../layouts/PageTemplate'

const PolicePage: React.FC = () => {
  const header = {
    title: 'Servers',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>PolicePage</PageTemplate>
    </>
  )
}

export default PolicePage

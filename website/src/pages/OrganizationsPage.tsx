import React from 'react'
import PageTemplate from '../layouts/PageTemplate'

const OrganizationsPage: React.FC = () => {
  const header = {
    title: 'Servers',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>OrganizationsPage</PageTemplate>
    </>
  )
}

export default OrganizationsPage

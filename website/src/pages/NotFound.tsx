import React from 'react'
import PageTemplate from '../layouts/PageTemplate'

const NotFoundPage: React.FC = () => {
  const header = {
    title: 'NotFound',
    description: 'programa de super poderes',
  }

  return <PageTemplate {...header}>NotFoundPage</PageTemplate>
}

export default NotFoundPage

import React from 'react'
import PageTemplate from '../layouts/PageTemplate'

const HomePage: React.FC = () => {
  const header = {
    title: 'Home',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>NotFoundPage</PageTemplate>
    </>
  )
}

export default HomePage

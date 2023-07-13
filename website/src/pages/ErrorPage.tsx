import React from 'react'
import { HTTPError } from '../types/http'
import PageTemplate from '../layouts/PageTemplate'

type ErrorPageProps = {
  error?: Error | HTTPError
}

const ErrorPage: React.FC<ErrorPageProps> = ({ error }) => {
  if (!error) return null

  const header = {
    title: 'Error',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>NotFoundPage</PageTemplate>
    </>
  )
}

export default ErrorPage

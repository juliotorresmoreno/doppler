import React from 'react'
import PageTemplate from '../layouts/PageTemplate'
import Crud from '../components/Crud'
import { Column } from '../models/column'
import { Col, Row } from 'reactstrap'

const columns: Column[] = [
  {
    code: 'id',
    title: 'Id',
    styles: {
      width: 100,
    },
  },
  {
    code: 'name',
    title: 'Server name',
  },
  {
    code: 'ipaddress',
    title: 'IP Address',
  },
  {
    code: 'organization',
    title: 'Organization',
  },
  {
    code: 'created_at',
    title: 'Created At',
  },
]

const ServersPage: React.FC = () => {
  const header = {
    title: 'Servers',
    description: 'Programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>
        <Crud columns={columns} />
      </PageTemplate>
    </>
  )
}

export default ServersPage

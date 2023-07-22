import React from 'react'
import PageTemplate from '../../layouts/PageTemplate'
import { Col, Row } from 'reactstrap'
import AddButton from './AddButton'

const ServersPage: React.FC = () => {
  const header = {
    title: 'Servers',
    description: 'Programa de super poderes',
  }

  return (
    <PageTemplate {...header}>
      <Row md={5}>
        <Col>
          <AddButton />
        </Col>
      </Row>
    </PageTemplate>
  )
}

export default ServersPage

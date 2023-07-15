import React from 'react'
import { Col, Row } from 'reactstrap'
import PlusCard from './PlusCard'
import FormCard from './FormCard'

const CrudBaseCard: React.FC = () => {
  return (
    <Row md={4}>
      <Col>
        <FormCard />
      </Col>
      <Col>
        <PlusCard />
      </Col>
    </Row>
  )
}

export default CrudBaseCard

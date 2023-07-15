import React, { useState } from 'react'
import { Card, CardBody, PlusCircleFill } from './styled'
import FormCard from './FormCard'

const PlusCard: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false)

  if (isOpen) {
    return <FormCard />
  }

  return (
    <Card onClick={() => setIsOpen(true)}>
      <CardBody align="center">
        <PlusCircleFill />
      </CardBody>
    </Card>
  )
}

export default PlusCard

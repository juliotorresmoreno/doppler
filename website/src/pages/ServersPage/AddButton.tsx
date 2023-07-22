import React, { useState } from 'react'
import ModalForm from './ModalForm'
import { Card, CardBody, PlusCircleFill } from './styled'

const AddButton: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <ModalForm isOpen={isOpen} toggle={() => setIsOpen(!isOpen)} />
      <Card onClick={() => setIsOpen(true)}>
        <CardBody align="center">
          <PlusCircleFill />
        </CardBody>
      </Card>
    </>
  )
}

export default AddButton

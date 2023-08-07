import React from 'react'
import { ModalBody } from './styled'
import { Modal } from 'reactstrap'
import FormData from './FormData'

type ModalFormProps = {
  id?: number
  isOpen: boolean
  toggle: () => void
}

const ModalForm: React.FC<ModalFormProps> = ({ id, isOpen, toggle }) => {
  return (
    <Modal isOpen={isOpen} toggle={toggle}>
      <ModalBody>
        <FormData toggle={toggle} />
      </ModalBody>
    </Modal>
  )
}

export default ModalForm

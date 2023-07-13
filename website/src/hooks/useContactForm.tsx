import React, { useState } from 'react'
import {
  Button,
  Col,
  Form,
  FormGroup,
  Input,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
  Row,
} from 'reactstrap'

const useContactForm = (): [React.FC, () => void] => {
  const [isOpen, setIsOpen] = useState(false)
  const toggle = () => setIsOpen(!isOpen)
  const Component: React.FC = () => (
    <Modal isOpen={isOpen} toggle={toggle}>
      <ModalHeader toggle={toggle}>Contact</ModalHeader>
      <ModalBody>
        <Form autoComplete="off">
          <Input
            autocomplete="false"
            name="hidden"
            type="text"
            class="hidden"
          />
          <FormGroup>
            <Label>Full name</Label>
            <Input autoComplete="off" name="fullname" type="text" />
          </FormGroup>
          <FormGroup>
            <Label>Email</Label>
            <Input autoComplete="off" name="email" type="email" />
          </FormGroup>
          <FormGroup>
            <Label>Subject</Label>
            <Input autoComplete="off" name="subject" type="text" />
          </FormGroup>
          <FormGroup>
            <Label>Description</Label>
            <Input
              autoComplete="off"
              name="description"
              type="textarea"
              style={{ height: 200 }}
            />
          </FormGroup>
          <Button color="primary">Submit</Button>
        </Form>
      </ModalBody>
    </Modal>
  )
  return [Component, toggle]
}

export default useContactForm

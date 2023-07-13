import React, { useState } from 'react'
import {
  Button,
  Form,
  FormGroup,
  Input,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
} from 'reactstrap'

const useSignInForm = (): [React.FC, () => void] => {
  const [isOpen, setIsOpen] = useState(false)
  const toggle = () => setIsOpen(!isOpen)
  const Component: React.FC = () => (
    <Modal isOpen={isOpen} toggle={toggle}>
      <ModalHeader toggle={toggle}>Sign In</ModalHeader>
      <ModalBody>
        <Form autoComplete="off">
          <FormGroup>
            <Label>Email</Label>
            <Input autoComplete='none' id='email' name="email" type="email" />
          </FormGroup>
          <FormGroup>
            <Label>Password</Label>
            <Input autoComplete='none' id='password' name="password" type="password" />
          </FormGroup>
          <Button color="primary">Submit</Button>
        </Form>
      </ModalBody>
    </Modal>
  )
  return [Component, toggle]
}

export default useSignInForm

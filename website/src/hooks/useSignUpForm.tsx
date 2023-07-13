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

const useSignUpForm = (): [React.FC, () => void] => {
  const [isOpen, setIsOpen] = useState(false)
  const toggle = () => setIsOpen(!isOpen)
  const Component: React.FC = () => (
    <Modal isOpen={isOpen} toggle={toggle}>
      <ModalHeader toggle={toggle}>Sign Up</ModalHeader>
      <ModalBody>
        <Form autoComplete="off">
          <FormGroup>
            <Row>
              <Col>
                <Label>Name</Label>
                <Input autoComplete="off" name="name" type="text" />
              </Col>
              <Col>
                <Label>Lastname</Label>
                <Input autoComplete="off" name="lastname" type="text" />
              </Col>
            </Row>
          </FormGroup>
          <FormGroup>
            <Label for="exampleEmail">Email</Label>
            <Input autoComplete="off" name="email" type="email" />
          </FormGroup>
          <FormGroup>
            <Row>
              <Col>
                <Label>Password</Label>
                <Input autoComplete="off" name="password" type="password" />
              </Col>
              <Col>
                <Label>Repeat password</Label>
                <Input
                  autoComplete="off"
                  name="repeat-password"
                  type="password"
                />
              </Col>
            </Row>
          </FormGroup>
          <Button color="primary">Submit</Button>
        </Form>
      </ModalBody>
    </Modal>
  )
  return [Component, toggle]
}

export default useSignUpForm

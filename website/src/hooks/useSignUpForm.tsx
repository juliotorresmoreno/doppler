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
import useInputForm from './useInputForm'
import config from '../config'

const useSignUpForm = (): [React.FC, () => void] => {
  const [isOpen, setIsOpen] = useState(false)
  const toggle = () => setIsOpen(!isOpen)

  const Component: React.FC = () => {
    const name = useInputForm(config.testUserName)
    const lastname = useInputForm(config.testUserLastname)
    const email = useInputForm(config.testUserEmail)
    const password = useInputForm(config.testUserPassword)
    const repeatpassword = useInputForm(config.testUserPassword)

    return (
      <Modal isOpen={isOpen} toggle={toggle}>
        <ModalHeader toggle={toggle}>Sign Up</ModalHeader>
        <ModalBody>
          <Form autoComplete="off">
            <FormGroup>
              <Row>
                <Col>
                  <Label>Name</Label>
                  <Input
                    autoComplete="off"
                    name="name"
                    type="text"
                    value={name.value}
                    onChange={name.onChange}
                  />
                </Col>
                <Col>
                  <Label>Lastname</Label>
                  <Input
                    autoComplete="off"
                    name="lastname"
                    type="text"
                    value={lastname.value}
                    onChange={lastname.onChange}
                  />
                </Col>
              </Row>
            </FormGroup>
            <FormGroup>
              <Label>Email</Label>
              <Input
                autoComplete="off"
                name="email"
                type="email"
                value={email.value}
                onChange={email.onChange}
              />
            </FormGroup>
            <FormGroup>
              <Row>
                <Col>
                  <Label>Password</Label>
                  <Input
                    autoComplete="off"
                    name="password"
                    type="password"
                    value={password.value}
                    onChange={password.onChange}
                  />
                </Col>
                <Col>
                  <Label>Repeat password</Label>
                  <Input
                    autoComplete="off"
                    name="repeat-password"
                    type="password"
                    value={repeatpassword.value}
                    onChange={repeatpassword.onChange}
                  />
                </Col>
              </Row>
            </FormGroup>
            <Button color="primary">Submit</Button>
          </Form>
        </ModalBody>
      </Modal>
    )
  }
  return [Component, toggle]
}

export default useSignUpForm

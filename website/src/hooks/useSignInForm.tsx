import React, { useState } from 'react'
import {
  Alert,
  Button,
  Form,
  FormGroup,
  Input,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
} from 'reactstrap'
import useInputForm from './useInputForm'
import config from '../config'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { signIn } from '../features/auth'

const useSignInForm = (): [React.FC, () => void] => {
  const [isOpen, setIsOpen] = useState(false)
  const toggle = () => setIsOpen(!isOpen)
  const appDispatch = useAppDispatch()
  const { error, isLoading } = useAppSelector((state) => state.auth)

  const Component: React.FC = () => {
    const email = useInputForm(config.testUserEmail)
    const password = useInputForm(config.testUserPassword)
    const onSubmit: React.FormEventHandler<HTMLFormElement> = (evt) => {
      evt.preventDefault()
      appDispatch(
        signIn({
          email: email.value,
          password: password.value,
        })
      ).then(function () {
        toggle()
      })
    }

    return (
      <Modal isOpen={isOpen} toggle={toggle}>
        <ModalHeader toggle={toggle}>Sign In</ModalHeader>
        <ModalBody>
          <Form onSubmit={onSubmit} autoComplete="off">
            <FormGroup>
              <Label>Email</Label>
              <Input
                value={email.value}
                onChange={email.onChange}
                autoComplete="none"
                name="email"
                type="email"
              />
            </FormGroup>
            <FormGroup>
              <Label>Password</Label>
              <Input
                value={password.value}
                onChange={password.onChange}
                autoComplete="none"
                name="password"
                type="password"
              />
            </FormGroup>
            {error ? <Alert color="danger">{error}</Alert> : null}
            <Button disabled={isLoading} color="primary">
              Submit
            </Button>
          </Form>
        </ModalBody>
      </Modal>
    )
  }
  return [Component, toggle]
}

export default useSignInForm

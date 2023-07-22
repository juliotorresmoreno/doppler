import React from 'react'
import { Button, TextArea } from './styled'
import { Form, FormGroup, Input, Label } from 'reactstrap'
import useInputForm from '../../hooks/useInputForm'

type FormDataProps = {
  id?: number
  toggle: () => void
}

const FormData: React.FC<FormDataProps> = ({ id, toggle }) => {
  const serverName = useInputForm('')
  const ipAddress = useInputForm('')
  const description = useInputForm('')
  const onSubmit: React.FormEventHandler<HTMLFormElement> = (evt) => {
    evt.preventDefault()
  }
  const onCancelClick: React.FormEventHandler<HTMLFormElement> = (evt) => {
    evt.preventDefault()
    toggle()
  }
  return (
    <Form onSubmit={onSubmit}>
      <FormGroup>
        <Label>Name</Label>
        <Input
          name="server-name"
          value={serverName.value}
          onChange={serverName.onChange}
        />
      </FormGroup>
      <FormGroup>
        <Label>IP Address</Label>
        <Input
          name="ip-address"
          value={ipAddress.value}
          onChange={ipAddress.onChange}
        />
      </FormGroup>
      <FormGroup>
        <Label>Description</Label>
        <TextArea
          name="description"
          value={description.value}
          onChange={description.onChange}
        />
      </FormGroup>
      {id ? (
        <Button color="primary" type="submit">
          Update
        </Button>
      ) : (
        <Button color="primary" type="submit">
          Create
        </Button>
      )}
      &nbsp;&nbsp;
      <Button color="secondary" onClick={onCancelClick}>
        Cancel
      </Button>
    </Form>
  )
}

export default FormData

import React from 'react'
import { Card, CardBody, TextArea, Button } from './styled'
import { Form, FormGroup, Input, Label } from 'reactstrap'

const FormCard: React.FC = () => {
  return (
    <Card>
      <CardBody>
        <Form>
          <FormGroup>
            <Label>Name</Label>
            <Input />
          </FormGroup>
          <FormGroup>
            <Label>IP Address</Label>
            <Input />
          </FormGroup>
          <FormGroup>
            <Label>Expose ports</Label>
            <TextArea />
          </FormGroup>
          <Button color="primary">Save</Button>&nbsp;
          <Button color="secondary">Cancel</Button>
        </Form>
      </CardBody>
    </Card>
  )
}

export default FormCard

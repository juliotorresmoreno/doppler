import React from 'react'
import {
  Button,
  Col,
  Form,
  FormGroup,
  Input,
  Label,
  Row,
} from 'reactstrap'
import PageTemplate from '../layouts/PageTemplate'

const SignInPage: React.FC = () => {
  const header = {
    title: 'SignIn',
    description: 'programa de super poderes',
  }

  return (
    <>
      <PageTemplate {...header}>
        <Row>
          <Col md={{ size: 8, offset: 2 }}>
            <Form>
              <FormGroup>
                <Label for="exampleEmail">Email</Label>
                <Input
                  id="exampleEmail"
                  name="email"
                  placeholder="with a placeholder"
                  type="email"
                />
              </FormGroup>
              <FormGroup>
                <Label for="examplePassword">Password</Label>
                <Input
                  id="examplePassword"
                  name="password"
                  placeholder="password placeholder"
                  type="password"
                />
              </FormGroup>
              <FormGroup check>
                <Input type="checkbox" /> <Label check>Check me out</Label>
              </FormGroup>
              <Button color='primary'>Submit</Button>
            </Form>
          </Col>
        </Row>
      </PageTemplate>
    </>
  )
}

export default SignInPage

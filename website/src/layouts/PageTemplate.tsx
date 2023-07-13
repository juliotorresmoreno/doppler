import React from 'react'
import Header from '../components/Header'
import AppNavBar from '../components/AppNavBar'
import { Col, Container, Row } from 'reactstrap'

type PageTemplateProps = {
  title: string
  description: string
} & React.PropsWithChildren

const PageTemplate: React.FC<PageTemplateProps> = ({
  title,
  description,
  children,
}) => {
  const header = {
    title,
    description,
  }

  return (
    <>
      <Container fluid>
        <AppNavBar>
          <Header {...header} />
        </AppNavBar>

        <Container>
          <main>{children}</main>
        </Container>
      </Container>
    </>
  )
}

export default PageTemplate

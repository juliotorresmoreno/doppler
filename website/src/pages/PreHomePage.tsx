import React from 'react'
import PageTemplate from '../layouts/PageTemplate'
import AppCarousel from '../components/AppCarousel'
import AppCard from '../components/AppCard'
import { Col, Row } from 'reactstrap'
import GoogleLoginButton from '../components/GoogleLoginButton'

const PreHomePage: React.FC = () => {
  const header = {
    title: 'PreHome',
    description: 'programa de super poderes',
  }

  return (
    <PageTemplate {...header}>
      <section>
        <AppCarousel />
      </section>

      <br />

      <section>
        <Row md={4}>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
        </Row>
      </section>

      <br />

      <section>
        <Row>
          <Col md={{ size: 9 }}>
            <h2 className="featurette-heading">
              First featurette heading.{' '}
              <span className="text-muted">It'll blow your mind.</span>
            </h2>
            <p className="lead">
              Donec ullamcorper nulla non metus auctor fringilla. Vestibulum id
              ligula porta felis euismod semper. Praesent commodo cursus magna,
              vel scelerisque nisl consectetur. Fusce dapibus, tellus ac cursus
              commodo.
            </p>
          </Col>
          <Col md={{ size: 3 }}>
            <img
              className="featurette-image img-fluid mx-auto"
              alt="500x500"
              style={{ width: '100%' }}
              src="https://picsum.photos/300/200"
              data-holder-rendered="true"
            />
          </Col>
        </Row>
      </section>

      <br />

      <section>
        <Row md={4}>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
          <Col>
            <AppCard />
          </Col>
        </Row>
      </section>

      <br />
      <section>
        <footer>
          <Row>
            <Col md={{ size: 8 }}>
              <p style={{ textAlign: 'justify' }}>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit. Morbi
                hendrerit finibus ornare. Nam quis consectetur odio, sit amet
                mollis dui. Vestibulum posuere magna ac quam varius tempus.
                Quisque neque est, pulvinar ac velit et, efficitur porttitor
                risus. Curabitur egestas mollis molestie. Nunc fermentum, mi sit
                amet tincidunt ullamcorper, metus neque iaculis orci, ac blandit
                nisl lacus semper est. In eu velit sed nisl sollicitudin
                venenatis non eu felis. Donec id porttitor nisi. Integer blandit
                ultrices dolor id dictum.
              </p>
            </Col>
            <Col>
              <p>La la la</p>
            </Col>
          </Row>
        </footer>
      </section>
    </PageTemplate>
  )
}

export default PreHomePage

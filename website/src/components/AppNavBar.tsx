import React, { useCallback, useState } from 'react'
import {
  Collapse,
  Navbar,
  NavbarToggler,
  NavbarBrand,
  Nav,
  NavItem,
  NavbarProps,
} from 'reactstrap'
import withSession from '../hoc/withSession'
import { ISession } from '../models/session'
import { Link } from 'react-router-dom'
import useSignInCanvas from '../hooks/useSignInForm'
import useSignUpCanvas from '../hooks/useSignUpForm'
import useContactForm from '../hooks/useContactForm'

type _AppNavBarProps = {
  session: ISession
} & NavbarProps &
  React.PropsWithChildren

const _AppNavBar: React.FC<_AppNavBarProps> = ({
  children,
  session,
  ...navBarProps
}) => {
  const [isOpenCollapse, setIsOpenCollapse] = useState(false)
  const toggleCollapse = useCallback(
    () => setIsOpenCollapse(!isOpenCollapse),
    []
  )
  const [SignInForm, toggleSignIn] = useSignInCanvas()
  const [SignUpForm, toggleSignUp] = useSignUpCanvas()
  const [ContactForm, toggleContact] = useContactForm()
  const onSignInClick: React.MouseEventHandler = (e) => {
    e.preventDefault()
    toggleSignIn()
  }
  const onSignUpClick: React.MouseEventHandler = (e) => {
    e.preventDefault()
    toggleSignUp()
  }
  const onContactClick: React.MouseEventHandler = (e) => {
    e.preventDefault()
    toggleContact()
  }

  return (
    <div>
      <SignInForm />
      <SignUpForm />
      <ContactForm />

      <Navbar light {...navBarProps} expand="md">
        <NavbarBrand href="/">{children}</NavbarBrand>

        <NavbarToggler onClick={toggleCollapse} />
        <Collapse isOpen={isOpenCollapse} navbar>
          {!session ? (
            <>
              <Nav className="me-auto" navbar>
                <NavItem>
                  <a className="nav-link" href="" onClick={onSignInClick}>
                    Sign In
                  </a>
                </NavItem>
                <NavItem>
                  <a className="nav-link" href="" onClick={onSignUpClick}>
                    Sign Up
                  </a>
                </NavItem>
              </Nav>
            </>
          ) : null}
        </Collapse>

        <Navbar>
          <a className="nav-link" href="/contact" onClick={onContactClick}>
            Contact
          </a>
        </Navbar>
      </Navbar>
    </div>
  )
}

const AppNavBar = withSession(_AppNavBar)

export default AppNavBar

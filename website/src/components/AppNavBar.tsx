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
import useContactForm from '../hooks/useContactForm'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import useSignInForm from '../hooks/useSignInForm'
import useSignUpForm from '../hooks/useSignUpForm'
import { Link } from 'react-router-dom'
import authSlice from '../features/auth'

type _AppNavBarProps = {} & NavbarProps & React.PropsWithChildren

const _AppNavBar: React.FC<_AppNavBarProps> = ({
  children,
  ...navBarProps
}) => {
  const session = useAppSelector((state) => state.auth.session)
  const [isOpenCollapse, setIsOpenCollapse] = useState(false)
  const appDispatch = useAppDispatch()
  const toggleCollapse = useCallback(
    () => setIsOpenCollapse(!isOpenCollapse),
    []
  )
  const [SignInForm, toggleSignIn] = useSignInForm()
  const [SignUpForm, toggleSignUp] = useSignUpForm()
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

  const onLogoutClick = () => {
    appDispatch(authSlice.actions.logout())
  }

  return (
    <div>
      <SignInForm />
      <SignUpForm />
      <ContactForm />

      <Navbar light {...navBarProps} expand="md">
        <NavbarBrand href="/">{children}</NavbarBrand>
        {!session ? (
          <>
            <NavbarToggler onClick={toggleCollapse} />
            <Collapse isOpen={isOpenCollapse} navbar>
              <Nav navbar>
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
            </Collapse>
          </>
        ) : (
          <>
            <NavbarToggler onClick={toggleCollapse} />
            <Collapse isOpen={isOpenCollapse} navbar>
              <Nav navbar>
                <NavItem>
                  <Link className="nav-link" to="/servers">
                    Servers
                  </Link>
                </NavItem>
                <NavItem>
                  <Link className="nav-link" to="/history">
                    History
                  </Link>
                </NavItem>
                <NavItem>
                  <Link className="nav-link" to="/organization">
                    Organization
                  </Link>
                </NavItem>
                <NavItem>
                  <Link className="nav-link" to="/police">
                    Police
                  </Link>
                </NavItem>
              </Nav>
            </Collapse>
          </>
        )}

        <Navbar>
          <Nav navbar>
            {!session ? (
              <NavItem>
                <a
                  className="nav-link"
                  href=""
                  onClick={onContactClick}
                >
                  Contact
                </a>
              </NavItem>
            ) : (
              <NavItem>
                <a href="/" className="nav-link" onClick={onLogoutClick}>
                  Logout
                </a>
              </NavItem>
            )}
          </Nav>
        </Navbar>
      </Navbar>
    </div>
  )
}

const AppNavBar = _AppNavBar

export default AppNavBar

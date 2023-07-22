import {
  Card as BSCard,
  CardBody as BSCardBody,
  Input as BSInput,
  Button as BSButton,
  ModalBody as BSModalBody,
  NavItem as BSNavItem,
} from 'reactstrap'
import { BsPlusCircleFill } from 'react-icons/bs'
import styled from 'styled-components'

export const Card = styled(BSCard)``

export const CardBody = styled(BSCardBody)`
  cursor: pointer;
`

export const PlusCircleFill = styled(BsPlusCircleFill).attrs({
  align: 'right',
})`
  font-size: 8rem;
  text-align: ${(props) => props.align};
`

export const Button = styled(BSButton)`
  width: 100px;
`

export const ModalBody = styled(BSModalBody)`
  height: 400;
`

export const TabPaneBody = styled.div`
  padding-top: 10px;
  height: 415px;
`

export const NavItem = styled(BSNavItem)`
  cursor: pointer;
`

export const TextArea = styled((props) => (
  <BSInput {...props} type="textarea" />
))`
  height: 150px;
`

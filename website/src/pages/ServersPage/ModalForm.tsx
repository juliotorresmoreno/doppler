import React, { useState } from 'react'
import { ModalBody, NavItem, TabPaneBody } from './styled'
import { Modal, Nav, NavLink, TabContent, TabPane } from 'reactstrap'
import FormData from './FormData'
import classNames from 'classnames'

type ModalFormProps = {
  id?: number
  isOpen: boolean
  toggle: () => void
}

const ModalForm: React.FC<ModalFormProps> = ({ id, isOpen, toggle }) => {
  const [tab, selectTab] = useState('1')

  return (
    <Modal isOpen={isOpen} toggle={toggle}>
      <ModalBody>
        <Nav tabs>
          <NavItem>
            <NavLink
              className={classNames({ active: tab === '1' })}
              onClick={() => selectTab('1')}
            >
              Basic
            </NavLink>
          </NavItem>
          <NavItem>
            <NavLink
              className={classNames({ active: tab === '2' })}
              onClick={() => selectTab('2')}
              disabled={!id}
            >
              Ports
            </NavLink>
          </NavItem>
          <NavItem>
            <NavLink
              className={classNames({ active: tab === '3' })}
              onClick={() => selectTab('3')}
              disabled={!id}
            >
              Permissions
            </NavLink>
          </NavItem>
        </Nav>
        <TabContent activeTab={tab}>
          <TabPane tabId="1">
            <TabPaneBody>
              <FormData toggle={toggle} />
            </TabPaneBody>
          </TabPane>
          <TabPane tabId="2">
            <TabPaneBody></TabPaneBody>
          </TabPane>
          <TabPane tabId="3">
            <TabPaneBody></TabPaneBody>
          </TabPane>
        </TabContent>
      </ModalBody>
    </Modal>
  )
}

export default ModalForm

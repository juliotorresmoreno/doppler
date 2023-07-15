import React from 'react'
import { Toast, ToastBody, ToastHeader } from 'reactstrap'

const Slot: React.FC = () => {
  return (
    <Toast style={{ width: '100%', height: 300, marginBottom: 20 }}>
      <ToastHeader>Reactstrap</ToastHeader>
      <ToastBody>
        This is a toast on a white background — check it out!
      </ToastBody>
    </Toast>
  )
}

export default Slot

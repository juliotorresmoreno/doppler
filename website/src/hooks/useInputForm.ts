import React, { useState } from 'react'

const useInputForm = (defaultValue: any = '') => {
  const [value, setValue] = useState(defaultValue)

  const onChange: React.ChangeEventHandler<HTMLInputElement> = (evt) => {
    setValue(evt.target.value)
  }

  return { value, setValue, onChange }
}

export default useInputForm

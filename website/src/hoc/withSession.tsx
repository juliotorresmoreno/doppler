import React from 'react'

type ResultProps = {
  [x: string | number | symbol]: any
} & React.PropsWithChildren

const withSession = function <T = any>(
  WrappedComponent: React.ComponentType<any>
) {
  const Result: React.FC<T & ResultProps> = (props) => {
    return <WrappedComponent {...props} />
  }

  return Result
}

export default withSession

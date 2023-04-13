import React, { useEffect, useState } from 'react'
import Loading from '../components/Loading'
import authSlice from '../features/auth'
import { useGetSession } from '../services/auth'
import { useAppDispatch, useAppSelector } from '../store/hooks'

type ResultProps = {
  [x: string | number | symbol]: any
} & React.PropsWithChildren

const withSession = function <T = any>(
  WrappedComponent: React.ComponentType<any>
) {
  const Result: React.FC<T & ResultProps> = (props) => {
    const session = useAppSelector((state: any) => state.auth.session)
    const [data, setData] = useState<any>(null)
    const { isLoading, get } = useGetSession()
    const dispatch = useAppDispatch()

    useEffect(() => {
      if (!session) return
      if (isLoading) return
      if (data) return
      get()
        .then(setData)
        .catch((err: Error) => {
          if (err.message === 'unauthorized')
            dispatch(authSlice.actions.logout())
        })
    }, [isLoading])

    if (isLoading) return <Loading />

    return <WrappedComponent {...props} />
  }

  return Result
}

export default withSession

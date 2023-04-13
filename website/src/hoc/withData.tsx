import React, { useEffect, useState } from 'react'
import Loading from '../components/Loading'
import authSlice from '../features/auth'
import ErrorPage from '../pages/ErrorPage'
import { useGetData } from '../services/api'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { useParams } from 'react-router'

type ResultProps = {
  [x: string | number | symbol]: any
} & React.PropsWithChildren

type WithDataArgs<T> = {
  WrappedComponent: React.ComponentType<any>
  url: string | ((params: any) => string)
}

const withData = function <T = any, S = any>(args: WithDataArgs<S>) {
  const { WrappedComponent, url } = args

  const Result: React.FC<T & ResultProps> = (props) => {
    const session = useAppSelector((state: any) => state.auth.session)
    const [data, setData] = useState<S | null>(null)
    const params = useParams()
    const safeUrl = typeof url === 'function' ? url(params) : url
    const { get, error, isLoading } = useGetData(safeUrl)
    const dispatch = useAppDispatch()

    useEffect(() => {
      if (isLoading) return
      if (data) return
      get()
        .then((data: S) => setData(data))
        .catch((err) => {
          if (err.message === 'unauthorized')
            dispatch(authSlice.actions.logout())
        })
    }, [isLoading, session])

    if (error) return <ErrorPage error={error} />
    if (data === null) return <Loading />

    return <WrappedComponent payload={data} {...props} />
  }
  return Result
}

export default withData

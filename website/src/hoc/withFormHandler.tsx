import React from 'react'
import { useApi } from '../services/api'
import { HTTPError } from '../types/http'

type WithFormHandlerProps<Result = any> = {
  onSuccess?: (payload: Result) => void
  onFailure?: (error: Error) => void
}

type WrappedComponentProps<Payload = any> = {
  isLoading: boolean
  onSubmit: (payload: Payload) => void
  errors: HTTPError | {}
}

export default function withFormHandler<Payload = any, Result = any>(
  WrappedComponent: React.ComponentType<WrappedComponentProps<Payload>>,
  method: string,
  url: string
) {
  return function WithFormHandler({ onSuccess, onFailure }: WithFormHandlerProps<Result>) {
    const { error, isLoading, apply } = useApi<Payload, Result>(method, url)

    const onSubmit = async (payload: Payload) => {
      try {
        const response = await apply(payload)

        onSuccess && onSuccess(response)
      } catch (error: any) {
        onFailure && onFailure(error)
      }
    }
    return (
      <WrappedComponent
        isLoading={isLoading}
        onSubmit={onSubmit}
        errors={error ?? {}}
      />
    )
  }
}

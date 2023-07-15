import React, { Fragment } from 'react'
import { Button, Table } from 'reactstrap'
import { Column } from '../../models/column'
import { BsPlusCircleFill } from 'react-icons/bs'
import styled from 'styled-components'

export type CrudProps = {
  columns: Column[]
}

const CellAdd = styled.td`
  border: 0;
`

const Crud: React.FC<CrudProps> = ({ columns }) => {
  return (
    <>
      <Table>
        <thead>
          <tr>
            {columns.map((column) => (
              <Fragment key={column.code}>
                <th style={column.styles}>{column.title}</th>
              </Fragment>
            ))}
          </tr>
        </thead>
        <tbody>
          <tr>
            <CellAdd>
              <a style={{ cursor: 'pointer' }} color="primary">
                <BsPlusCircleFill /> Add
              </a>
            </CellAdd>
          </tr>
        </tbody>
      </Table>
    </>
  )
}

export default Crud

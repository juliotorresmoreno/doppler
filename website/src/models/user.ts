export type IUser = {
  id: number
  email: string
  name: string
  lastname: string
}
export class User {
  id: number
  email: string
  name: string
  lastname: string

  constructor(data: IUser) {
    this.id = data.id
    this.email = data.email
    this.name = data.name
    this.lastname = data.lastname
  }

  getFullName() {
    return this.name + ' ' + this.lastname
  }
}

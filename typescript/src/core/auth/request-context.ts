export interface Credentials {
  accessKeyId: string;
  accessKeySecret: string;
  securityToken?: string;
}

export class RequestContext {
  #credentials: Credentials;
  #signDate?: string;

  constructor(credentials: Credentials, signDate?: string) {
    this.#credentials = credentials;
    this.#signDate = signDate;
  }

  getCredentials(): Credentials {
    return this.#credentials;
  }

  getSignDate(): string | undefined {
    return this.#signDate;
  }

  setSignDate(signDate: string): void {
    this.#signDate = signDate;
  }
}

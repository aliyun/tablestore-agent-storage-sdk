export class OTSClientError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'OTSClientError';
  }
}

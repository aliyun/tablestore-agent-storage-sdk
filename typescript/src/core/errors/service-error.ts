export class OTSServiceError extends Error {
  readonly statusCode: number;
  readonly requestId?: string;
  readonly code?: string;

  constructor(message: string, options: { statusCode: number; requestId?: string; code?: string }) {
    super(message);
    this.name = 'OTSServiceError';
    this.statusCode = options.statusCode;
    this.requestId = options.requestId;
    this.code = options.code;
  }
}

export interface ErrorDetails {
  [key: string]: unknown;
}

export interface SerializedError {
  name: string;
  message: string;
  statusCode: number;
  code: string;
  details?: ErrorDetails;
}

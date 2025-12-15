'use server'

import {
  DataSourceResponse,
  DatabricksCatalog,
  DatabricksSchema,
  DatabricksTable,
  CreateDataSourceRequest,
  BaseResponse
} from "@/types";
import { fetchApi, createErrorResponse } from "./utils";

/**
 * Fetches all data sources from the backend API.
 * @returns Promise with data source array or error
 */
export async function getDataSources(): Promise<BaseResponse<DataSourceResponse[]>> {
  try {
    const response = await fetchApi<BaseResponse<DataSourceResponse[]>>(`/datasources`);

    if (!response) {
      throw new Error("Failed to get data sources");
    }

    return {
      message: "Data sources fetched successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<DataSourceResponse[]>(error, "Error getting data sources");
  }
}

/**
 * Creates a new DataSource.
 * @param request The create data source request
 * @returns Promise with created data source or error
 */
export async function createDataSource(request: CreateDataSourceRequest): Promise<BaseResponse<DataSourceResponse>> {
  try {
    const response = await fetchApi<BaseResponse<DataSourceResponse>>(`/datasources`, {
      method: 'POST',
      body: JSON.stringify(request),
    });

    if (!response) {
      throw new Error("Failed to create data source");
    }

    return {
      message: response.message || "Data source created successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<DataSourceResponse>(error, "Error creating data source");
  }
}

/**
 * Fetches available Databricks catalogs.
 * @returns Promise with catalog array or error
 */
export async function getDatabricksCatalogs(): Promise<BaseResponse<DatabricksCatalog[]>> {
  try {
    const response = await fetchApi<BaseResponse<DatabricksCatalog[]>>(`/databricks/catalogs`);

    if (!response) {
      throw new Error("Failed to get Databricks catalogs");
    }

    return {
      message: "Catalogs fetched successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<DatabricksCatalog[]>(error, "Error getting Databricks catalogs");
  }
}

/**
 * Fetches schemas for a specific Databricks catalog.
 * @param catalog The catalog name
 * @returns Promise with schema array or error
 */
export async function getDatabricksSchemas(catalog: string): Promise<BaseResponse<DatabricksSchema[]>> {
  try {
    const response = await fetchApi<BaseResponse<DatabricksSchema[]>>(`/databricks/catalogs/${encodeURIComponent(catalog)}/schemas`);

    if (!response) {
      throw new Error("Failed to get Databricks schemas");
    }

    return {
      message: "Schemas fetched successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<DatabricksSchema[]>(error, "Error getting Databricks schemas");
  }
}

/**
 * Fetches tables for a specific Databricks catalog and schema.
 * @param catalog The catalog name
 * @param schema The schema name
 * @returns Promise with table array or error
 */
export async function getDatabricksTables(catalog: string, schema: string): Promise<BaseResponse<DatabricksTable[]>> {
  try {
    const response = await fetchApi<BaseResponse<DatabricksTable[]>>(`/databricks/schemas/${encodeURIComponent(catalog)}/${encodeURIComponent(schema)}/tables`);

    if (!response) {
      throw new Error("Failed to get Databricks tables");
    }

    return {
      message: "Tables fetched successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<DatabricksTable[]>(error, "Error getting Databricks tables");
  }
}

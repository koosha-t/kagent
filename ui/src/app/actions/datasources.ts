'use server'

import { DataSourceResponse } from "@/types";
import { fetchApi, createErrorResponse } from "./utils";
import { BaseResponse } from "@/types";

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

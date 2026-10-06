import { ActionIcon, Tooltip } from "@mantine/core";
import { Column } from "@tanstack/react-table";
import { ArrowsDownUp, SortAscending, SortDescending } from "phosphor-react";

interface SortToggleProps {
  column: Column<any, unknown>;
}

// SortToggle cycles a column between unsorted, ascending and descending. It
// sits next to the header's filter control, which owns clicks on the header.
export function SortToggle({ column }: SortToggleProps) {
  const sorted = column.getIsSorted();
  const label =
    sorted === "asc"
      ? "Sorted ascending"
      : sorted === "desc"
        ? "Sorted descending"
        : "Sort";

  return (
    <Tooltip label={label} openDelay={500}>
      <ActionIcon
        size="sm"
        variant={sorted ? "light" : "subtle"}
        color={sorted ? "brand" : "gray"}
        aria-label={label}
        onClick={column.getToggleSortingHandler()}>
        {sorted === "asc" ? (
          <SortAscending weight="bold" size={14} />
        ) : sorted === "desc" ? (
          <SortDescending weight="bold" size={14} />
        ) : (
          <ArrowsDownUp weight="bold" size={14} />
        )}
      </ActionIcon>
    </Tooltip>
  );
}

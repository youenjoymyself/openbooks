import {
  Button,
  Group,
  Indicator,
  Loader,
  ScrollArea,
  Table,
  Text,
  Tooltip
} from "@mantine/core";
import { useElementSize, useMergedRef } from "@mantine/hooks";
import {
  createColumnHelper,
  FilterFn,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getSortedRowModel,
  Row,
  SortingFn,
  SortingState,
  useReactTable
} from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";
import { MagnifyingGlass, User } from "phosphor-react";
import { useMemo, useRef, useState } from "react";
import { useSelector } from "react-redux";
import { useGetServersQuery } from "../../state/api";
import { BookDetail } from "../../state/messages";
import { sendDownload } from "../../state/stateSlice";
import { RootState, useAppDispatch } from "../../state/store";
import FacetFilter, {
  ServerFacetEntry,
  StandardFacetEntry
} from "./Filters/FacetFilter";
import { TextFilter } from "./Filters/TextFilter";
import { SortToggle } from "./SortToggle";
import { useTableStyles } from "./styles";

const columnHelper = createColumnHelper<BookDetail>();

const stringInArray: FilterFn<any> = (
  row,
  columnId: string,
  filterValue: string[] | undefined
) => {
  if (!filterValue || filterValue.length === 0) return true;

  return filterValue.includes(row.getValue<string>(columnId));
};

const sizeUnits: Record<string, number> = {
  "": 1,
  "K": 1024,
  "M": 1024 ** 2,
  "G": 1024 ** 3,
  "T": 1024 ** 4
};

// parseSize converts sizes like "8.44MB" or "775.69 KiB" to bytes. Returns -1
// when the size is missing or unparseable.
const parseSize = (size: string): number => {
  const match = /^\s*([\d.,]+)\s*([KMGT]?)i?B?\s*$/i.exec(size ?? "");
  if (!match) return -1;
  const value = parseFloat(match[1].replace(/,/g, ""));
  return isNaN(value) ? -1 : value * sizeUnits[match[2].toUpperCase()];
};

const bySize: SortingFn<BookDetail> = (a, b, columnId) =>
  parseSize(a.getValue(columnId)) - parseSize(b.getValue(columnId));

// withTitle shows the full text on hover, since cells are clamped to one line.
const withTitle = (value: string) => <span title={value}>{value}</span>;

interface BookTableProps {
  books: BookDetail[];
}

export default function BookTable({ books }: BookTableProps) {
  const { classes, cx, theme } = useTableStyles();
  const { data: servers } = useGetServersQuery(null);

  const { ref: elementSizeRef, height, width } = useElementSize();
  const virtualizerRef = useRef();
  const mergedRef = useMergedRef(elementSizeRef, virtualizerRef);

  const columns = useMemo(() => {
    const cols = (cols: number) => (width / 12) * cols;
    return [
      columnHelper.accessor("server", {
        header: (props) => (
          <FacetFilter
            placeholder="Server"
            column={props.column}
            table={props.table}
            Entry={ServerFacetEntry}
          />
        ),
        cell: (props) => {
          const online = servers?.includes(props.getValue());
          return (
            <Text
              size={12}
              weight="normal"
              color="dark"
              style={{ marginLeft: 20 }}>
              <Tooltip
                position="top-start"
                label={online ? "Online" : "Offline"}>
                <Indicator
                  zIndex={0}
                  position="middle-start"
                  offset={-16}
                  size={6}
                  color={online ? "green.6" : "gray"}>
                  {props.getValue()}
                </Indicator>
              </Tooltip>
            </Text>
          );
        },
        size: cols(1),
        enableColumnFilter: true,
        filterFn: stringInArray
      }),
      columnHelper.accessor("author", {
        header: (props) => (
          <TextFilter
            icon={<User weight="bold" />}
            placeholder="Author"
            column={props.column}
            table={props.table}
          />
        ),
        cell: (props) => withTitle(props.getValue()),
        size: cols(2),
        enableColumnFilter: false
      }),
      columnHelper.accessor("title", {
        header: (props) => (
          <TextFilter
            icon={<MagnifyingGlass weight="bold" />}
            placeholder="Title"
            column={props.column}
            table={props.table}
          />
        ),
        cell: (props) => withTitle(props.getValue()),
        minSize: 20,
        size: cols(6),
        enableColumnFilter: false
      }),
      columnHelper.accessor("format", {
        header: (props) => (
          <FacetFilter
            placeholder="Format"
            column={props.column}
            table={props.table}
            Entry={StandardFacetEntry}
          />
        ),
        size: cols(1),
        enableColumnFilter: false,
        filterFn: stringInArray
      }),
      columnHelper.accessor("size", {
        header: "Size",
        size: cols(1),
        enableColumnFilter: false,
        sortingFn: bySize
      }),
      columnHelper.display({
        header: "Download",
        size: cols(1),
        enableColumnFilter: false,
        enableSorting: false,
        cell: ({ row }) => (
          <DownloadButton book={row.original.full}></DownloadButton>
        )
      })
    ];
  }, [width, servers]);

  const [sorting, setSorting] = useState<SortingState>([]);

  const table = useReactTable({
    data: books,
    columns: columns,
    state: { sorting },
    onSortingChange: setSorting,
    enableFilters: true,
    columnResizeMode: "onChange",
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues()
  });

  const { rows: tableRows } = table.getRowModel();
  const filteredCount = table.getFilteredRowModel().rows.length;

  const rowVirtualizer = useVirtualizer({
    count: tableRows.length,
    getScrollElement: () => virtualizerRef.current,
    estimateSize: () => 50,
    overscan: 10
  });

  const virtualItems = rowVirtualizer.getVirtualItems();

  const paddingTop =
    virtualItems.length > 0 ? virtualItems?.[0]?.start || 0 : 0;
  const paddingBottom =
    virtualItems.length > 0
      ? rowVirtualizer.getTotalSize() -
        (virtualItems?.[virtualItems.length - 1]?.end || 0)
      : 0;

  return (
    <>
      {filteredCount !== books.length && (
        <Text size="sm" color="dimmed" sx={{ width: "100%" }} mb={4}>
          Showing {filteredCount} of {books.length} results.
        </Text>
      )}
      <ScrollArea
        viewportRef={mergedRef}
        className={classes.container}
        type="hover"
        scrollbarSize={6}
        styles={{ thumb: { ["&::before"]: { minWidth: 4 } } }}
        offsetScrollbars={false}>
        <Table highlightOnHover verticalSpacing="sm" fontSize="xs">
          <thead className={classes.head}>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className={classes.headerCell}
                    style={{
                      width: header.getSize()
                    }}>
                    <Group noWrap spacing={4}>
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                      {header.column.getCanSort() && (
                        <SortToggle column={header.column} />
                      )}
                    </Group>
                    <div
                      onMouseDown={header.getResizeHandler()}
                      onTouchStart={header.getResizeHandler()}
                      className={cx(classes.resizer, {
                        ["isResizing"]: header.column.getIsResizing()
                      })}
                    />
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {paddingTop > 0 && (
              <tr>
                <td style={{ height: `${paddingTop}px` }} />
              </tr>
            )}
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const row = tableRows[
                virtualRow.index
              ] as unknown as Row<BookDetail>;
              return (
                <tr key={row.id} style={{ height: 50 }}>
                  {row.getVisibleCells().map((cell) => {
                    return (
                      <td key={cell.id}>
                        <Text lineClamp={1} color="dark">
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )}
                        </Text>
                      </td>
                    );
                  })}
                </tr>
              );
            })}
            {paddingBottom > 0 && (
              <tr>
                <td style={{ height: `${paddingBottom}px` }} />
              </tr>
            )}
          </tbody>
        </Table>
      </ScrollArea>
    </>
  );
}

function DownloadButton({ book }: { book: string }) {
  const dispatch = useAppDispatch();

  const isInFlight = useSelector((state: RootState) =>
    state.state.inFlightDownloads.includes(book)
  );

  // Prevent requesting the same book twice. Re-enabled once the download
  // finishes or fails, so failed downloads can be retried.
  const onClick = () => {
    if (isInFlight) return;
    dispatch(sendDownload(book));
  };

  return (
    <Button
      compact
      size="xs"
      radius="sm"
      onClick={onClick}
      sx={{ fontWeight: "normal", width: 80 }}>
      {isInFlight ? (
        <Loader variant="dots" color="gray" />
      ) : (
        <span>Download</span>
      )}
    </Button>
  );
}

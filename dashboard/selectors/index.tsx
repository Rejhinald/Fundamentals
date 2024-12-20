import { createSelector } from "reselect";
import _ from "lodash";
import dayjs from "dayjs";

export const actionItemsSelector = state => state && state.ActionItems.actionItems;
export const sortSelector = state => state && state.ActionItems.sortParams;
export const searchSelector = state => state && state.Activities.searchKey;

const createdDate = item => dayjs.unix(item.CreatedAt).format("YYYY-MM-DD");


export const getSortedActionItems = createSelector(
    actionItemsSelector,
    sortSelector,
    searchSelector,
    (actionItems, sort, key) => {
        if (key) {
            actionItems = actionItems.filter(activity => {
                if (activity?.SearchKey?.toLowerCase()?.includes(key.toLowerCase())) return activity;
            });
        }

        //? Sorted on the backend using scanForwardIndex
        //? We just need to groupBy here on the frontend
        const groupedActionItems = _.groupBy(actionItems, createdDate);
        return groupedActionItems;
    }
);
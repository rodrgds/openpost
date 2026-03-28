import { type DateValue } from '@internationalized/date';

class UIState {
	isComposeOpen = $state(false);
	composeInitialDate = $state<DateValue | undefined>(undefined);
	refreshCounter = $state(0);

	openCompose(date?: DateValue) {
		this.composeInitialDate = date;
		this.isComposeOpen = true;
	}

	closeCompose() {
		this.isComposeOpen = false;
	}

	triggerRefresh() {
		this.refreshCounter++;
	}
}

export const ui = new UIState();

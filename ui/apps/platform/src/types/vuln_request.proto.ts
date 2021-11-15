import { SlimUser } from './user.proto';

export type RequestComment = {
    id: string;
    user: SlimUser;
    message: string;
    createdAt: string;
};
